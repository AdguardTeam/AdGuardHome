const express = require('express')

const app = express()

app.use(express.static(__dirname))

console.log('Open http://localhost:4000/ to examine the API spec')
app.listen(4000)
