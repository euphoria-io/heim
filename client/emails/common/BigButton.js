import React from 'react'

import { Item, Link, Text, textDefaults } from '../email'


export default function BigButton(props) {
  return (
    <Item align="center" cellPadding={24}>
      <Link color="white" textDecoration="none" href={props.href} style={{
        display: 'inline-block',
        background: props.color,
        padding: '22px 30px',
        borderRadius: '4px',
      }}>
        <Text {...textDefaults} fontSize={24} fontWeight="bold" color="white">{props.children}</Text>
      </Link>
    </Item>
  )
}

BigButton.propTypes = {
  href: React.PropTypes.string.isRequired,
  color: React.PropTypes.string,
  children: React.PropTypes.node,
}
