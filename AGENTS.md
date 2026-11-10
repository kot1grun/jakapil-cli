# AGENT.md — руководство для агентов в этой кодовой базе

Этот файл экономит время агента (и человека): здесь собран минимум, который нужно
знать, прежде чем менять код jakapil-cli. Он отвечает на три вопроса: **что** это
за проект, **зачем**, и **как** его собирать/тестировать/проверять. Если правил
хватает — правь код по образцу существующих команд (`cmd/<service>`), а не по
шаблонам других проектов.

---

## 1. Обзор (THE WHAT и THE WHY)

**jakapil-cli** — «неофициальный» CLI-клиент на Go для облачных сервисов Selectel.
Он использует уже существующие Go SDK от Selectel (`mks-go`, `craas-go`,
`go-selvpcclient/v5`) и `gophercloud` как фундамент для авторизации и работы с
частью сервисов. Задумка — «универсальная» CLI, похожая на openstack CLI, с
поддержкой многих сервисов Selectel. Пул проектов/аккаунтов Selectel бывает разным,
поэтому **endpoint'ы сервисов не хардкодим**: их резолвим из Keystone service catalog.

Монолитная структура: **один Go-модуль**, одно приложение (`main.go`), внутри —
иерархия команд с общими внутренними пакетами.

### Закреплённые факты актуального состояния (проверено на текущем коде)
- Заголовок лицензии/автора во всех `.go`: `/* Copyright © 2026 Kot1gruN */`.
- Комментарии и README дублируются на русском и английском; код-комментарии —
  английские. Отвечать/писать можно на языке пользователя, код комментируется
  по-английски.
- Раньше поддерживались только MKS и CRaaS; сейчас добавлены `vpc`, `compute`,
  `selproject`, `authenticate`, `onboard`, `config`. Новые сервисы добавляются
  по мере появления SDK.

---

## 2. Технологический стек

- **Язык/сборка**: Go (`go.mod: go 1.27.2`).
- **CLI**: `spf13/cobra` (команды и флаги).
- **Конфиг/окружение**: `spf13/viper` + переменные `OS_*` (openrc), `SLT_KS_TOKEN`.
- **Keystone/облако**: `gophercloud/gophercloud/v2`, `gophercloud/gophercloud/v1`
  (используется некоторыми SDK Selectel), `gophercloud/utils/v2` (clientconfig,
  чтение clouds.yaml — используется в `authenticate`).
- **SDK Selectel**:
  - `selectel/mks-go` — Managed Kubernetes (MKS);
  - `selectel/craas-go` — Container Registry as a Service (CRaaS);
  - `selectel/go-selvpcclient/v5` — Cloud Platform Projects (Resell) API;
- **Вывод**: `rivo/tview` + `gdamore/tcell` (рендер таблиц в терминале),
  `go.yaml.in/yaml/v3`, `mattn/go-runewidth` (ширина CJK-символов).
- **Прочее**: `schollz/progressbar/v3` (прогресс), `golang.org/x/term` (пароли).

**Версионирование**: module path — `jakapil-cli` (короткий, не `github.com/...`).
Импорты внутри проекта — `jakapil-cli/cmd/...`, `jakapil-cli/internal/...`.

---

## 3. Структура проекта

```
main.go                     точка входа: только cmd.Execute()
cmd/                        все CLI-команды (монолит, по одной на сервис)
  root.go                   корневая команда; регистрирует все деревья команд
  <service>/<service>.go    New<Service>Command(): persistent-флаги + подкоманды
  <service>/client/         сборка клиента gophercloud/SDK для сервиса
  <service>/<object>/       подкоманды (list/create/update/delete/...)
internal/                   переиспользуемая логика, не являющаяся командой
  authentication/           сохранённые токены (tokenfile), их загрузка
  keystone/                 авторизация + discovery endpoint'ов через каталог
  output/                   рендер: Print (json/yaml), PrintTable, RenderRegionsBestEffort
  parallel/                 параллельный fan-out по регионам (только чтение)
assets/                     картинки/материалы README
bin/                        артефакт сборки (gitignored)
```

### Паттерн команды (важно соблюдать при добавлении новых)
1. `cmd/<service>/<service>.go` — `New<Service>Command() *cobra.Command`:
   - persistent-флаги: `--token/-t` (`$SLT_KS_TOKEN`), `--auth-url/-e`
     (`$OS_AUTH_URL`, дефолт `https://cloud.api.selcloud.ru/identity/v3`),
     при необходимости `--region` (`$OS_REGION_NAME`), `--debug` (печать
     каталога и endpoint'ов в stderr), `--output/-o` (table|json|yaml);
   - `Run` по умолчанию печатает help;
   - регистрирует подкоманды из `cmd/<service>/<object>/`.
2. `cmd/<service>/client/client.go` — сборка клиента:
   - константы флагов и env-переменных (`FlagToken`, `EnvToken` и т.д.);
   - `resolve*Credentials(cmd)` — приоритет: флаг → сохранённый токен →
     env → дефолт. Важно: флаг с ненулевым дефолтом должен выигрывать у env
     только когда пользователь его задал явно (`cmd.Flags().Changed(...)`).
   - `New<Service>Client(cmd)` — для сервисов через `internal/keystone.Authenticate`
     (TokenID) + `openstack.New...V2(provider, ...)` или `DiscoverEndpoint`.
3. `cmd/<service>/<object>/*.go` — `new<list/create/...>Command()`, каждая — с
   локальными флагами под команду. Логика из `RunE` выносится в `run*`-функции
   (`runList`, `runCreate`...), которые принимают `*cobra.Command` (для вывода и
   флагов) и клиент — это делает их юнит-тестируемыми через мок-сервер.
4. Рендер результатов: `internal/output` (Print / PrintTable / RenderRegionsBestEffort).

### Внутренние пакеты — что делают
- **`internal/authentication`**: файл токена (по умолчанию
  `~/.config/jakapil-cli/token`, права `0600`), формат YAML со scope
  (`project` / `account`), время выдачи/истечения. `LoadUsableStoredToken()` —
  проектный токен (для MKS/CRaaS); `LoadUsableAccountToken()` — аккаунтный
  (для Resell/`selproject`). `ErrTokenExpired` — сигнал устаревшего токена.
- **`internal/keystone`**: `Authenticate` (TokenID, без scope → token lookup +
  каталог), `DiscoverEndpoint`/`DiscoverRegions`, `ResolveEndpointFromProvider`,
  `printDebug` — дамп каталога и выбранного endpoint'а при `--debug`.
- **`internal/output`**: унифицированный вывод. `Print` — json/yaml/plain;
  `PrintTable` — таблицы rivo/tview (с автошириной и уважением к runewidth);
  `RenderRegionsBestEffort` — агрегированный вывод по регионам (колонка `Region`
  в table, группировка в json/yaml).
- **`internal/parallel`**: `parallel.Regions(ctx, regions, fn)` — конкурентный,
  идемпотентный GET по регионам; возвращает упорядоченный `[]Result` (один на
  регион), ошибка одного региона не роняет остальные.

---

## 4. HOW: сборка, тесты, проверка

### Быстрые команды
```bash
make build          # go build -o bin/jakapil-cli .
make test           # go test ./...
make vet            # go vet ./...
make lint           # go vet + golangci-lint, если установлен
make lint-check     # требует golangci-lint и запускает его
make clean          # rm -rf bin/
# или напрямую:
go build -o bin/ .
go test ./...
```

### Требуемые инструменты
- Go в версии из `go.mod` (сейчас `go 1.27.2`).
- **GNU Make** (в Makefile есть не-POSIX конструкции; на macOS ставится через
  `brew install make` как `gmake`).
- `golangci-lint` необязателен (используется только при `make lint`/`lint-check`).

### Верификация изменений (минимальная перед сдачей)
1. `go build ./...` — всё компилируется.
2. `go vet ./...` и, если доступен, `golangci-lint run ./...`.
3. `go test ./...` — все тесты зелёные.
4. `gofmt -l .` — не должно быть файлов вне форматирования; `gofmt -w` перед коммитом.

### Как пишутся тесты здесь (особенность проекта)
- Юнит-тесты на команды живут рядом (например `cmd/selproject/selproject_test.go`)
  и используют **`net/http/httptest`** для мока сервера в том же тесте.
- `testClient(t)` (см. `selproject_test.go`) строит реальный клиент SDK, но с
  локальным мок-`ServeMux`, регистрируя `/auth/tokens` (возвращает каталог с
  endpoint'ом Resell) и `/resell/v2/projects` (нужные методы). Хелперы `testCommand`
  регистрируют все флаги команды и подменяют stdout на `strings.Builder`.
- Тестовая компиляция требует импорт SDK-клиентов (например
  `selvpcclient/clients/services`), поэтому тест-пакет `package selproject`
  (не `_test`) — так хелперы видят неэкспортируемые `runList` и др.
- Тесты на `internal/authentication` (tokenfile) опираются на временные файлы.
- Ошибки API (например `policy_does_not_allow_this_request`) проверяются через
  `errors.As`/`errors.Is`, чтобы убедиться, что обёртки (`fmt.Errorf("%w", ...)`)
  сохраняют исходную ошибку.

### Неочевидные трюки и подводные камни
- **Два gophercloud** в зависимостях (v1 и v2) — не путать пути импорта.
  SDK Selectel могут тащить v1, а общий код CLI использует `/v2`.
- **Endpoint без trailing slash ломает запросы**: URL версии (`.../v1`,
  `.../v2`) валидируется клиентами; `internal/keystone` сам срезает `/`, а
  `selproject/client.normalizeEndpoint` — добавляет, чтобы gophercloud мог
  резолвить относительные URL.
- **Флаги с ненулевым дефолтом vs env**: `identityEndpoint` в проверке
  использует `cmd.Flags().Changed(FlagEndpoint)`, иначе `$OS_AUTH_URL` никогда
  не подхватится.
- **Секреты в выводе**: токены и пароли маскируются (`internal/output`,
  `cmd/config.renderEnv`); `authenticate` выводит значения токенов только для
  скриптов/`authenticate token issue`. Никогда не печатай содержимое
  `~/.config/jakapil-cli/token` в логи/ответы.
- **Скоуп токена критичен**: MKS/CRaaS требуют **project**-скоуп
  (`--project-scope`, дефолт); **Resell/selproject** требует **account**-скоуп
  (`--account-scope`). Загрузчики токенов различают их — не заменяй
  `LoadUsableAccountToken` на `LoadUsableStoredToken` для `selproject`.

### Как проверять вручную
```bash
# собрать
make build
# токены
./bin/jakapil-cli authenticate token issue            # project (по умолчанию)
./bin/jakapil-cli authenticate token issue --account-scope   # для selproject
./bin/jakapil-cli authenticate token list
# сервисы
./bin/jakapil-cli mks cluster list   --region ru-6
./bin/jakapil-cli vpc network list   --region ru-6
./bin/jakapil-cli selproject list    --region ru-6          # ждёт account-токен
# + --debug  для дампа каталога / endpoint
./bin/jakapil-cli mks cluster list --region ru-6 --debug
```
Подсказка: endpoint'ы Selectel могут отличаться между пулами/регионами — при
«странных» 403/404 гляди `--debug` вывод и сверяйся с
`https://docs.selectel.ru/api/urls/` (пул-специфичные хосты вида
`<pool>.cloud.api.selcloud.ru/...`).

---

## 5. Что делать при добавлении нового сервиса (чек-лист)
1. Убедись, что есть актуальный Go SDK/клиент от Selectel; добавь его в `go.mod`.
2. Создай `cmd/<service>/<service>.go` с `New<Service>Command()`, persistent-флаги
   по образцу `mks.go`/`vpc.go`, подкоманды — в `cmd/<service>/<object>/`.
3. Создай `cmd/<service>/client/client.go` с `resolve*Credentials` и
   `New<Service>Client`; не хардкодь endpoint — бери из каталога через
   `internal/keystone`.
4. Зарегистрируй команду в `cmd/root.go` (`rootCmd.AddCommand(...)`).
5. Напиши юнит-тесты с `httptest` моком (изучи `selproject_test.go`).
6. `go build ./... && go vet ./... && go test ./...` и `gofmt -w`.

---

*Обновляй этот файл при добавлении значимых пакетов, команд или изменении
процесса сборки/проверки.*