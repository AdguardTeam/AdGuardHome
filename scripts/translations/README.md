## Twosky intergration script

### Usage

```
npm install
TWOSKY_URI=<API URI> TWOSKY_CLIENT_ID=<PROJECT ID> node download.js
TWOSKY_URI=<API URI> TWOSKY_CLIENT_ID=<PROJECT ID> node upload.js
```

After download you'll find the output locales in the `client/src/__locales/` folder.

Examples:
```
TWOSKY_URI=https://twosky.example/api/v1 TWOSKY_CLIENT_ID=adguardhome node download.js
TWOSKY_URI=https://twosky.example/api/v1 TWOSKY_CLIENT_ID=adguardhome node upload.js
```
