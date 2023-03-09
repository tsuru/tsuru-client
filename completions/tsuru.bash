_tsuru() {
  curr_line="${COMP_LINE:0:$COMP_POINT}"
  suggestions=$( AUTOCOMPLETE_CURRENT_LINE="${curr_line}" "${TSURU_PATH:-tsuru}" )
  COMPREPLY=( $(compgen -W "${suggestions}") )
}

complete -F _tsuru tsuru
