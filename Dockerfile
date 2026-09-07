# ----- Builder -----

# base GO image
FROM golang:1.24.3
# Set working directory inside container
WORKDIR /app

# Cache deps first
COPY go.mod go.sum ./
RUN go mod download

# templ CLI for codegen
RUN GOBIN=/usr/local/bin go install github.com/a-h/templ/cmd/templ@v0.3.924

# Copy from local project to container
COPY . .

# dev: generate then run
CMD ["sh","-lc","/usr/local/bin/templ generate && /usr/local/go/bin/go run ./cmd/app"]