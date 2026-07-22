# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /iconfirm .

# ---- run stage ----
FROM alpine:3.20
WORKDIR /app
COPY --from=build /iconfirm /app/iconfirm
# โฟลเดอร์เก็บรูปที่อัปโหลด (serve ที่ /uploads)
RUN mkdir -p /app/uploads
EXPOSE 8080
CMD ["/app/iconfirm"]
