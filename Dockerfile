# Установка модулей и тесты
FROM golang:1.25.5 AS modules

ADD go.mod go.sum /m/
RUN cd /m && go mod download

# RUN make test

# Сборка приложения
FROM golang:1.25.5 AS builder

COPY --from=modules /go/pkg /go/pkg

# Пользователь без прав
RUN useradd -u 10001 runner-runner

RUN mkdir -p /noted-runner
RUN mkdir -p /noted/codes/kernels
RUN chown -R runner-runner:runner-runner /noted/codes
ADD . /noted-runner
WORKDIR /noted-runner

# Сборка
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
    go build -o ./bin/noted-runner ./cmd/api

# Запуск в пустом контейнере
FROM gcr.io/distroless/cc-debian12

# Копируем пользователя без прав с прошлого этапа
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder --chown=runner-runner:runner-runner /noted/codes /noted/codes
# Запускаем от имени этого пользователя
USER runner-runner

COPY --from=builder /noted-runner/bin/noted-runner /noted-runner

CMD ["/noted-runner"]
