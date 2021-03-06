package main

import (
  "flag"
  "fmt"
  "net/http"
  "net/url"
  "pk/api"

  "code.google.com/p/goauth2/oauth"
)

type cmd struct {
  name  string
  run   func() error
  usage func() string
  flags *flag.FlagSet
}

var commands = []*cmd{
  cmdLogin,

  cmdKeyAdd,
  cmdKeyRemove,
  cmdKeysList,

  cmdProjectCreate,
  cmdProjectsList,
  cmdProjectDelete,
}

var client *api.PKClient

func main() {
  var w = flag.Bool("w", false, "prints list of commands")

  flag.Usage = func() {
    fmt.Println("Usage: pk <command> [options] \n")

    fmt.Println("Commands: \n")
    for _, c := range commands {
      fmt.Printf("  %16s  %s\n", c.name, c.usage())
    }
    fmt.Println()
    fmt.Println("Run 'pk help [command]' for more information.")
    fmt.Println()
    // flag.PrintDefaults()
  }

  flag.Parse()
  if *w {
    for _, c := range commands {
      fmt.Printf("%s ", c.name)
    }
    fmt.Println()
    return
  }

  var err error
  authorize(false)

  // save access token
  if flag.NArg() == 0 {
    flag.Usage()
    return
  }

  if flag.Arg(0) == "help" {
    command := findCommand(flag.Arg(1))
    if command == nil {
      flag.PrintDefaults()
      return
    }

    if command.flags == nil {
      fmt.Println("No usage for", command.name)
    } else {
      command.flags.PrintDefaults()
    }

    return
  }

  command := findCommand(flag.Arg(0))
  if command == nil {
    flag.PrintDefaults()
    return
  }

  if command.flags != nil {
    command.flags.Parse(flag.Args()[1:])
  }
  err = tryWithReauth(command.run)
  if err != nil {
    fmt.Printf("%s error: %s\n", flag.Arg(0), err)
  }
}

func findCommand(name string) *cmd {
  for _, c := range commands {
    if c.name == name {
      return c
    }
  }
  return nil
}

func tryWithReauth(f func() error) error {
  err := f()
  needsReauth := false
  switch err.(type) {
  case *url.Error:
    switch err.(*url.Error).Err.(type) {
    case oauth.OAuthError:
      fmt.Println("Access token has expired; please log in again.")
      needsReauth = true
    }
  case *api.APIError:
    if err.(*api.APIError).Code == http.StatusUnauthorized {
      fmt.Println("Bad access token; please log in again.")
      needsReauth = true
    }
  }

  if needsReauth {
    authorize(true)
    return f()
  }

  return err
}
