import React from 'react'

import Box from './Box'
import Item from './Item'


// inspired by bits and pieces of http://htmlemailboilerplate.com
export default React.createClass({
  propTypes: {
    title: React.PropTypes.string.isRequired,
    bgcolor: React.PropTypes.string,
    cellPadding: React.PropTypes.number,
    cellSpacing: React.PropTypes.number,
    children: React.PropTypes.node,
  },

  render() {
    // default nested 600px wide outer table container (see http://templates.mailchimp.com/development/html/)
    return (
      <html xmlns="http://www.w3.org/1999/xhtml">
        <head>
          <meta httpEquiv="Content-Type" content="text/html; charset=utf-8" />
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
