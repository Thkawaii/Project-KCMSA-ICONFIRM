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
# tzdata      = ให้ Asia/Bangkok ใช้ได้จริง (รอบส่งอีเมลรายสัปดาห์อิงโซนเวลานี้)
# ca-certificates = ให้ต่อ TLS ไปยัง smtp.office365.com / Microsoft Graph ได้
RUN apk add --no-cache tzdata ca-certificates
COPY --from=build /iconfirm /app/iconfirm
# โลโก้ที่ฝังบนหัวจดหมายของอีเมลแจ้งเตือนรายสัปดาห์
COPY --from=build /app/assets /app/assets
# โฟลเดอร์เก็บรูปที่อัปโหลด (serve ที่ /uploads)
RUN mkdir -p /app/uploads
EXPOSE 8080
CMD ["/app/iconfirm"]
