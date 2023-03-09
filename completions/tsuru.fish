function __fish_tsuru
  complete -c tsuru --erase
  complete -fc tsuru -a "(__fish_tsuru)"
  set -q TSURU_PATH; or set -x TSURU_PATH tsuru
  set -x AUTOCOMPLETE_CURRENT_LINE $(commandline -cp)
  printf "$($TSURU_PATH)"
end

complete -fc tsuru -a "(__fish_tsuru)"
