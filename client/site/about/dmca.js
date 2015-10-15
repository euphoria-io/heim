var fs = require('fs')
var React = require('react')

var common = require('../common')


module.exports = (
  <common.MainPage title="euphoria: copyright policy" nav={<common.PolicyNav selected="dmca" />}>
    <common.Markdown className="policy" content={fs.readFileSync(__dirname + '/dmca.md', 'utf8')} />
  </common.MainPage>
)
