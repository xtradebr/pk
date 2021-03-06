package main

import (
  "flag"
  "fmt"
  "os"
  "strings"
  "time"

  "code.google.com/p/go.crypto/ssh"
)

var argSSHPubKeyPath string

func init() {
  cmdKeyAdd.flags.StringVar(&argSSHPubKeyPath, "key-path", "", "path to an ssh-key to upload")
}

var cmdKeyAdd = &cmd{
  name: "add-key",
  run: func() (err error) {
    existingKeyFingerprints := make(map[string]bool)
    listKeyResp, listKeyErr := client.ListKeys()
    if listKeyErr == nil {
      for _, key := range listKeyResp.Keys {
        existingKeyFingerprints[key.Fingerprint] = true
      }
    }

    var key ssh.PublicKey
    if argSSHPubKeyPath != "" {
      key, _, err = sshReadPubKey(argSSHPubKeyPath)
    } else {
      keys := getSSHKeys(existingKeyFingerprints)
      if len(keys) == 0 && shouldContinue("No SSH keys to add. Create a new key?") {
        createSSHKey()
        keys = getSSHKeys(existingKeyFingerprints)
      }

      if len(keys) == 0 {
        return fmt.Errorf("No SSH keys to add.")
      }

      key, err = pickSSHKey(keys, os.Stdout)
    }

    if err != nil {
      return fmt.Errorf("refusing to upload: %s", err)
    }

    var keyName string
    keyName = getText("Name this key [optional]")
    if keyName == "" {
      keyName = time.Now().Format("2006-01-02")
    }

    keyStr := string(ssh.MarshalAuthorizedKey(key))
    err = client.UploadKey(keyName, strings.TrimSpace(keyStr))
    if err != nil {
      return err
    }

    fmt.Printf("Added key '%s'.\n", keyName)
    return nil
  },
  flags: flag.NewFlagSet("add-key", flag.ExitOnError),
  usage: func() string {
    return "add an ssh key to your account"
  },
}

var cmdKeysList = &cmd{
  name: "list-keys",
  run: func() error {
    resp, err := client.ListKeys()
    if err != nil {
      return err
    }

    for _, key := range resp.Keys {
      fmt.Printf("%-20s  %s  %s\n", key.Name, key.Fingerprint, key.Preview)
    }

    return nil
  },
  usage: func() string {
    return "list added ssh keys"
  },
}

var cmdKeyRemove = &cmd{
  name: "remove-key",
  run: func() error {
    resp, err := client.ListKeys()
    if err != nil {
      return err
    }

    for i, key := range resp.Keys {
      fmt.Printf("[ %d ] %-20s %s\n", i+1, key.Name, key.Preview)
    }

    choice, err := pick("key", len(resp.Keys))
    if err != nil {
      return err
    }

    chosenKey := resp.Keys[choice]
    _, err = client.DeleteKey(chosenKey.Fingerprint)
    if err != nil {
      return err
    }

    fmt.Println("Removed key", chosenKey.Preview)
    return nil
  },
  usage: func() string {
    return "diassocate an ssh key"
  },
}
