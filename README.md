# user-stats

Simple analytics service.

Uses MySQL DB to store data. Both services are containerized.

How to build and run on localhost:

```
sudo apt update
sudo apt install software-properties-common
sudo apt-add-repository --yes --update ppa:ansible/ansible
sudo apt install ansible
git submodule update --init
cd ansible
ansible-playbook -l localhost playbooks/setup-local.yml
```

Works on localhost:8090 by default.
Configuration can be changed in config.yml.

