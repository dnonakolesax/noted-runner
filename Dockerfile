# Установка модулей и тесты
FROM golang:1.25.5 AS modules

ADD go.mod go.sum /m/
RUN cd /m && go mod download

# RUN make test

# Сборка приложения
FROM golang:1.25.5 AS builder

COPY --from=modules /go/pkg /go/pkg

# Пользователь без прав
# RUN useradd -u 10001 runner-runner

RUN mkdir -p /noted-runner
RUN mkdir -p /noted/codes/kernels
RUN mkdir -p /var/run
# RUN chown -R runner-runner:runner-runner /noted/codes
# RUN chown -R runner-runner:runner-runner /var/run
ADD . /noted-runner
WORKDIR /noted-runner

# Сборка
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
    go build -o ./bin/noted-runner ./cmd/api

CMD ["./bin/noted-runner"]
