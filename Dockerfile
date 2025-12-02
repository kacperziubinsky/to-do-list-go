FROM ubuntu:latest

RUN apt-get update && apt-get install -y golang-go git

WORKDIR /app
COPY . .


RUN go build -o main .

CMD ["./main"]

# Expose port 8080
EXPOSE 8080

# To build and run the Docker container:
# docker build -t go-task-app .
# docker run -p 8080:8080 go-task-app

