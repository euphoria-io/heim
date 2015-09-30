var React = require('react')
var marked = require('marked')


var HEIM_PREFIX = process.env.HEIM_PREFIX || ''
var heimURL = module.exports.heimURL = function(href) {
  return HEIM_PREFIX + href
}

var Page = module.exports.Page = React.createClass({
  render: function() {
    return (
      <html>
      <head>
        <meta charSet="utf-8" />
        <title>{this.props.title}</title>
        <link rel="icon" id="favicon" href={heimURL('/static/favicon.png')} sizes="32x32" />
        <link rel="icon" href={heimURL('/static/favicon-192.png')} sizes="192x192" />
        <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no" />
        <link rel="stylesheet" type="text/css" id="css" href={heimURL('/static/site.css')} />
      </head>
      <body className={this.props.className}>
        {this.props.children}
      </body>
      </html>
    )
  },
})

var Header = module.exports.Header = React.createClass({
  render: function() {
    return (
      <header>
        <div className="container">
          <a className="logo" href={heimURL('/')}>euphoria</a>
          <a className="start-chatting" href={heimURL('/room/welcome/')} target="_blank">start chatting &raquo;</a>
        </div>
      </header>
    )
  },
})

var Footer = module.exports.Footer = React.createClass({
  render: function() {
    return (
      <footer>
        <div className="container">
          <a href="https://github.com/euphoria-io/heim" target="_blank">source code</a>
          <a href={heimURL('/about/values')} target="_blank">values</a>
          <a href={heimURL('/about/conduct')} target="_blank">code of conduct</a>
          <a href="http://andeuphoria.tumblr.com/" target="_blank">blog</a>
          <a href="mailto:hi@euphoria.io" target="_blank">contact</a>
        </div>
      </footer>
    )
  },
})

module.exports.MainPage = React.createClass({
  render: function() {
    return (
      <Page className="page" title={this.props.title}>
        <Header />
        <div className="container main">
          {this.props.children}
        </div>
        <Footer />
      </Page>
    )
  },
})

module.exports.Markdown = React.createClass({
  render: function() {
    return (
      <div className={this.props.className} dangerouslySetInnerHTML={{__html: marked(this.props.content)}} />
    )
  },
})
