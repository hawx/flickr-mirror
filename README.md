# flickr-mirror

This probably isn't useful yet. Currently it can index a folder structure like

```
data.json
photos/
  5453534534/
    data.json
    photo_o.jpg
    photo_z.jpg
  ...
sets/
  413133334/
    data.json
```

which is currently produced by [hawx/hall-of-mirrors][].

Once the data has been indexed you can spin up a local server to view your photos.

[hawx/hall-of-mirrors]: https://github.com/hawx/hall-of-mirrors/
