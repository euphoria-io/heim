import React from 'react'

import { Item, Link, Text, textDefaults } from '../email'


export default React.createClass({
  propTypes: {
    href: React.PropTypes.string.isRequired,
    color: React.PropTypes.string,
    children: React.PropTypes.node,
  },

  render() {
    return (
      <Item align="center" cellPadding={24}>
        <Link color="white" textDecoration="none" href={this.props.href} style={{
          display: 'inline-block',
          background: this.props.color,
          padding: '22px 30px',
          borderRadius: '4px',
        }}>
          <Text {...textDefaults} fontSize={24} fontWeight="bold" color="white">{this.props.children}</Text>
        </Link>
      </Item>
    )
  },
})
