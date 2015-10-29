import React from 'react'

import { Box, Item, Link, Image } from '../email'


export default React.createClass({
  propTypes: {
    logo: React.PropTypes.string.isRequired,
    padding: React.PropTypes.number,
    children: React.PropTypes.node,
  },

  getDefaultProps() {
    return {
      padding: 7,
    }
  },

  render() {
    return (
      <Item align="center">
        <Link href="{{.SiteURL}}">
          <Image src={'{{.File `' + this.props.logo + '`}}'} width={67} height={90} />
        </Link>
        <Box width="600" cellPadding={2} bgcolor="white" style={{
          borderBottom: '3px solid #ccc',
          borderRadius: '10px',
          padding: this.props.padding,
        }}>
          {this.props.children}
        </Box>
      </Item>
    )
  },
})
