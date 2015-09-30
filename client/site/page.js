var React = require('react')


module.exports.render = function(pageComponent) {
  var doctype = '<!doctype html>'
  return doctype + React.renderToStaticMarkup(pageComponent)
}
