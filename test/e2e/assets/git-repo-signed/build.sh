#!/bin/bash

set -e

# https://www.gnupg.org/documentation/manuals/gnupg/Unattended-GPG-key-generation.html
export GNUPGHOME="$(mktemp -d)"
gpg --batch --gen-key <<EOF
Key-Type: 1
Subkey-Type: 1
Name-Real: Trusted Trusted
Name-Email: trusted@k14s.io
Expire-Date: 0
EOF
gpg --batch --gen-key <<EOF
Key-Type: 1
Subkey-Type: 1
Name-Real: Stranger Stranger
Name-Email: stranger@k14s.io
Expire-Date: 0
EOF
trusted_id=$(gpg --list-secret-keys --keyid-format long trusted@k14s.io |grep sec|awk -F'[/ ]' '{print $5}')
stranger_id=$(gpg --list-secret-keys --keyid-format long stranger@k14s.io |grep sec|awk -F'[/ ]' '{print $5}')

rm -rf keys/
mkdir keys/
gpg --armor --export $trusted_id > keys/trusted.pub
gpg --armor --export $stranger_id > keys/stranger.pub

rm -rf git-meta/
mkdir git-meta/

rm -rf git-repo/
mkdir git-repo/
cd git-repo/

git init .
git config user.email "git@k14s.io"
git config user.name "Git Git"

# unsigned
echo "unsigned-commit" > file.txt
git add .
git commit -m 'unsigned-commit-msg'
git log -1 --format=format:%H > ../git-meta/unsigned-commit.txt

git tag unsigned-tag -m 'unsigned-tag-msg'

# signed by trusted
git config user.signingkey "$trusted_id"

git tag -s signed-trusted-tag-for-unsigned-commit -m 'signed-trusted-tag-for-unsigned-commit-msg'

echo "signed-trusted-commit" > file.txt
git add .
git commit -S -m 'signed-trusted-commit-msg'
git log -1 --format=format:%H > ../git-meta/signed-trusted-commit.txt

git tag -s signed-trusted-tag -m 'signed-trusted-tag-msg'


git submodule add https://github.com/vmware-tanzu/carvel-vendir
git add .
git commit -S -m 'signed-trusted-commit-msg'
git log -1 --format=format:%H > ../git-meta/signed-trusted-commit-git-submodule.txt

git tag -s git-submodule -m 'git-submodule'

# signed by stranger
git config user.signingkey "$stranger_id"

echo "signed-stranger-commit" > file.txt
git add .
git commit -S -m 'signed-stranger-commit-msg'
git log -1 --format=format:%H > ../git-meta/signed-stranger-commit.txt

git tag -s signed-stranger-tag -m 'signed-stranger-tag-msg'

# meta
git log --oneline > ../git-meta/commits.txt
git tag > ../git-meta/tags.txt

cd ../
tar czvf asset.tgz git-repo/ keys/ git-meta/
