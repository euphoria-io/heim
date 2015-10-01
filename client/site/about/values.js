var fs = require('fs')
var React = require('react')

var common = require('../common')


module.exports = (
  <common.MainPage title="euphoria: values" nav={<common.PolicyNav selected="values" />}>
    <common.Markdown className="policy" content={fs.readFileSync(__dirname + '/values.md', 'utf8')} />
  </common.MainPage>
)
