# Shell Agnostic Autocomplete

## Install on bash

Choose one of this options:

- place the file inside `/etc/bash_completion.d/`
  ```
  cp completions/tsuru.bash /etc/bash_completion.d/tsuru.sh
  ```

  or

- place the file in your location of choice and source that file from `.bashrc`
  ```
  cp completions/tsuru.bash ~/.tsuru-completion.sh
  grep -qxF '. ~/.tsuru-completion.sh' ~/.bashrc || echo "\n. ~/.tsuru-completion.sh" >> ~/.bashrc
  ```

then, reopen your shell.

## Install on zsh

Choose one of this options:

- place the file inside your framework (eg: oh-my-zsh) completions folder

  or

- place the file in your location of choice and source that file from `.zshrc`
  ```
  cp completions/tsuru.zsh ~/.tsuru-completion.zsh
  grep -qxF '. ~/.tsuru-completion.zsh' ~/.zshrc || echo "\n. ~/.tsuru-completion.zsh" >> ~/.zshrc
  ```

then, reopen your shell.

## Install on fish

Place the file at `~/.config/fish/completions/tsuru.fish`:
```
cp completions/tsuru.fish ~/.config/fish/completions/tsuru.fish
```

then, reopen your shell.
