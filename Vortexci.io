dist: trusty
sudo: false

language: php

php:
    - 5.5
    - 5.6
    - 7.0
    - 7.1
    - 7.2
    - 7.3

cache:
    directories:
    - vendor

before_install:
    - travis_retry composer self-update

install:
    - travis_retry composer install --no-interaction --prefer-source
