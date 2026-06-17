## Профилирование памяти и оптимизации

### Бенчмарки

Бенчмарки находятся в:
- `internal/repository/bench_test.go`
- `internal/handlers/bench_test.go`

Запуск:
```
go test ./internal/handlers/ -bench=. -benchmem -count=1 -run='^$'
go test ./internal/repository/ -bench=. -benchmem -count=1 -run='^$'
```

### Оптимизации

1. **`internal/repository/memstorage.go`** — заменены `sync.Mutex.Lock` на `sync.RWMutex.RLock` в методах `GetValue`, `GetGauge`, `GetCounter`; `GetAllGauge`/`GetAllCounter` теперь возвращают безопасные копии карт под `RLock`.
2. **`internal/repository/memstorage.go`** — `Dump` использует `json.Marshal` вместо `json.MarshalIndent` (нет лишних аллокаций на отступы).
3. **`internal/handlers/handlers.go`** — `ListHandler` использует `io.WriteString` вместо `w.Write([]byte(body.String()))` — устраняет лишнее копирование строки.
4. **`internal/compress/gzip.go`** — `sync.Pool` для переиспользования `gzip.Writer` между запросами вместо `gzip.NewWriter` при каждом запросе.

### Результат: `pprof -top -diff_base=profiles/base.pprof profiles/result.pprof`

```
File: handlers.test
Type: alloc_space
Showing nodes accounting for 1571.17MB, 20.89% of 7521.41MB total
      flat  flat%   sum%        cum   cum%
 -199.77MB  2.66%         -199.77MB        handlers.(*MetricsHandler).ListHandler
 -136.06MB  1.81%         -136.06MB        encoding/json.(*Decoder).refill
 -134.04MB  1.78%         -134.04MB        net/textproto.MIMEHeader.Set
 -101.10MB  1.34%         -101.10MB        strings.(*Builder).WriteString
  -97.53MB  1.30%          -97.53MB        encoding/json.NewDecoder
  -45.06MB  0.60%          -45.06MB        bytes.growSlice
  -25.00MB  0.33%         -309.10MB        handlers.(*MetricsHandler).JSONUpdateHandler
  -20.50MB  0.27%          -20.50MB        net/http.Header.Clone
  -16.50MB  0.22%          -25.50MB        encoding/json.(*decodeState).object
  -15.00MB  0.20%          -15.00MB        bytes.NewReader
  -13.54MB  0.18%          -13.54MB        reflect.growslice
   -7.00MB  0.09%           -9.00MB        encoding/json.(*decodeState).literalStore
```

Отрицательные значения подтверждают снижение аллокаций.

