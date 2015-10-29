import React from 'react'

import { Item, Box } from '../email'


export default React.createClass({
  propTypes: {
    children: React.PropTypes.node,
  },

  render() {
    return (
      <Item>
        <Box cellSpacing={20} width="100%" bgcolor="white" style={{borderBottom: '3px solid #ccc'}}>
          {this.props.children}
        </Box>
      </Item>
    )
  },
})
