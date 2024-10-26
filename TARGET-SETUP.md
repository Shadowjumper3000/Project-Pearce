# Setup Guide for OpenSSH and Generating an SSH Keypair

## Prerequisites

- Ensure you have `OpenSSH` installed on your local machine.
- You should have access to the servers you want to manage.

## Step 1: Install OpenSSH

### On Ubuntu/Debian

```sh
sudo apt update
sudo apt install openssh-server
```

```sh
ssh-keygen -t rsa -b 4096
```

```sh
ssh-copy-id your_username@192.168.1.10
```