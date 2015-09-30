var fs = require('fs')
var React = require('react')

var common = require('../common')


module.exports = (
  <common.MainPage title="euphoria: conduct">
    <common.Markdown className="policy" content={fs.readFileSync(__dirname + '/conduct.md', 'utf8')} />
  </common.MainPage>
)
