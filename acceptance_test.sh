#!/bin/zsh
# shellcheck disable=SC2046
test $(curl localhost:8002/api/add"?a=1&b=2") = '{"sum":3}'