var _ = require('lodash')
var React = require('react')

require('./inject-react-attributes')


// inspired by bits and pieces of http://htmlemailboilerplate.com
module.exports.Email = React.createClass({
  render: function() {
    // default nested 600px wide outer table container (see http://templates.mailchimp.com/development/html/)
    return (
      <html xmlns="http://www.w3.org/1999/xhtml">
        <head>
          <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
          <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
          <title>{this.props.title}</title>
        </head>
        <body style={{
          width: '100%',
          margin: '0',
          padding: '0',
          WebkitTextSizeAdjust: '100%',
          MsTextSizeAdjust: '100%',
        }}>
          <Box width="100%" height="100%" bgcolor={this.props.bgcolor}>
            <Item align="center" valign="top">
              <Box width="600" cellPadding={this.props.cellPadding} cellSpacing={this.props.cellSpacing}>
                {this.props.children}
              </Box>
            </Item>
          </Box>
        </body>
      </html>
    )
  },
})

var Box = module.exports.Box = React.createClass({
  getDefaultProps: function() {
    return {
      cellPadding: 0,
      cellSpacing: 0,
      border: '0',
      align: 'left',
      valign: 'top',
    }
  },

  render: function() {
    return (
      <table align={this.props.align} valign={this.props.valign} cellPadding={this.props.cellPadding} cellSpacing={this.props.cellSpacing} border={this.props.border} bgcolor={this.props.bgcolor} width={this.props.width} height={this.props.height} style={this.props.style}>
        {this.props.children}
      </table>
    )
  },
})

var Item = module.exports.Item = React.createClass({
  render: function() {
    return (
      <tr>
        <td align={this.props.align} valign={this.props.valign} bgcolor={this.props.bgcolor} style={this.props.style}>
          {this.props.children}
        </td>
      </tr>
    )
  },
})

module.exports.Text = React.createClass({
  getDefaultProps: function() {
    return {
      fontFamily: 'sans-serif',
      fontSize: 14,
      color: '#000',
    }
  },

  render: function() {
    return (
      <span style={_.merge({
        fontFamily: this.props.fontFamily,
        fontSize: this.props.fontSize,
        fontWeight: this.props.fontWeight,
        lineHeight: this.props.lineHeight !== null ? this.props.lineHeight : this.props.fontSize,
        color: this.props.color,
      }, this.props.style)}>{this.props.children}</span>
    )
  },
})

module.exports.Link = React.createClass({
  getDefaultProps: function() {
    return {
      textDecoration: 'underline',
    }
  },

  render: function() {
    return (
      <a href={this.props.href} target="_blank" style={_.merge({
        color: this.props.color,
        textDecoration: this.props.textDecoration,
      }, this.props.style)}>{this.props.children}</a>
    )
  },
})

module.exports.Image = React.createClass({
  render: function() {
    return (
      <img src={this.props.src} width={this.props.width} height={this.props.height} style={_.merge({
        display: 'block',
        outline: 'none',
        border: 'none',
        textDecoration: 'none',
      }, this.props.style)}>{this.props.children}</img>
    )
  },
})

module.exports.renderEmail = function(emailComponent) {
  var doctype = '<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">'
  return doctype + React.renderToStaticMarkup(emailComponent)
}
