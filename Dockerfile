FROM ubuntu:latest

RUN apt-get update && apt-get install -y golang-go git

WORKDIR /app
COPY . .


RUN go build -o main .

CMD ["./main"]

EXPOSE 8080



