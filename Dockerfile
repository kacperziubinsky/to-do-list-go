FROM ruby:3.2.2


RUN apt-get update -qq && apt-get install -y nodejs sqlite3 libsqlite3-dev

WORKDIR /app


COPY Gemfile Gemfile.lock ./


RUN bundle install


COPY . .

CMD ["rails", "server", "-b", "0.0.0.0"]