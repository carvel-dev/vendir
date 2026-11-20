# :bookmark_tabs:Scoop guidelines

[Scoop](https://scoop.sh/) is a command-line installer for Windows.

# :rocket:Get started

For now, to manually and locally install `vendir` :

```
git clone https://github.com/vmware-tanzu/carvel-vendir.git
cd carvel-vendir
export TARGET_VERSION=0.21.1
sed 's/$VENDIR_VERSION/'"$TARGET_VERSION"'/g' vendir.json.template > vendir.json
scoop install vendir
```

# :point_right:To do

To get this from a central repo, it is highly recommended to create a 
dedicated [Scoop Bucket](https://github.com/lukesampson/scoop/wiki/Buckets).

