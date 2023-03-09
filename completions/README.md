# Shell Agnostic Autocomplete

## Install on bash

Choose one of this options:

- place the file inside `/etc/bash_completion.d/`
  ```
  cp completions/tsuru.bash /etc/bash_completion.d/tsuru.sh
  ```

  or

- place the file in your location of choice (e.g. `~/.tsuru-completion.sh`)
  and source that file from .bashrc
  ```
  cp completions/tsuru.bash ~/.tsuru-completion.sh
  echo '. ~/.tsuru-completion.sh' >> ~/.bashrc
  ```

then, reopen your shell.

## Install on zsh

If you use oh-my-zsh, place the file at `~/.oh-my-zsh/completions/_tsuru`:
```
cp completions/tsuru.zsh ~/.oh-my-zsh/completions/_tsuru
```

If you use plain zsh, place the file at `~/.config/zsh/completions/_tsuru`:
```
cp completions/tsuru.zsh ~/.config/zsh/completions/_tsuru
```

then, reopen your shell.

## Install on fish

Place the file at `~/.config/fish/completions/tsuru.fish`:
```
cp completions/tsuru.fish ~/.config/fish/completions/tsuru.fish
```

then, reopen your shell.
