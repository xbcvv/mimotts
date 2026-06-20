FROM node:22-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

FROM golang:1.24-alpine AS backend
WORKDIR /src/backend
COPY backend/go.mod ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/frontend/dist ./static
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/mimotts .

FROM alpine:3.22
WORKDIR /app
RUN addgroup -S app && adduser -S app -G app && mkdir -p /app/data
COPY --from=backend /out/mimotts /app/mimotts
COPY --from=backend /src/backend/static /app/static
RUN chown -R app:app /app
USER app
ENV ADDR=:7117 DATA_DIR=/app/data
EXPOSE 7117
CMD ["/app/mimotts"]
