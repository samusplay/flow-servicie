
# ---------- Etapa 1: compilar ----------

FROM golang:1.27-alpine AS build

WORKDIR /src



# Primero solo las dependencias (se reutiliza la caché si no cambian)

COPY go.mod go.sum* ./

RUN go mod download



# Luego el código

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /flow .



# ---------- Etapa 2: imagen final, solo el ejecutable ----------

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app



COPY --from=build /flow /app/flow



ENV PORT=5007

EXPOSE 5007



ENTRYPOINT ["/app/flow"]

