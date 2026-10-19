# ---------- Etapa 1: compilar ----------
FROM golang:1.27-alpine AS build
WORKDIR /src

# primero solo las dependencias (se reutiliza la caché si no cambian)
COPY go.mod go.sum* ./
RUN go mod download

# luego el código
COPY . .
RUN CGO_ENABLED=0 go build -o /flow .

# ---------- Etapa 2: imagen final, solo el ejecutable ----------
FROM alpine:3.20
COPY --from=build /flow /flow
EXPOSE 5007
CMD ["/flow"]