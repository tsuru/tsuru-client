_tsuru(){
  compadd -- $(AUTOCOMPLETE_CURRENT_LINE="$LBUFFER" "${TSURU_PATH:-tsuru}")
}
compdef _tsuru tsuru
