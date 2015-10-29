import React from 'react'

import { Item } from '../email'


export default React.createClass({
  propTypes: {
    children: React.PropTypes.node,
  },

  render() {
    return (
      <Item style={{paddingLeft: '20px'}}>
        {this.props.children}
      </Item>
    )
  },
})
