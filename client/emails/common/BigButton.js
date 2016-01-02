import React from 'react'

import { Item, A, Span, textDefaults } from 'react-html-email'


export default function BigButton(props) {
  return (
    <Item align="center" cellPadding={24}>
      <A color="white" textDecoration="none" href={props.href} style={{
        display: 'inline-block',
        background: props.color,
        padding: '22px 30px',
        borderRadius: '4px',
      }}>
        <Span {...textDefaults} fontSize={24} fontWeight="bold" color="white">{props.children}</Span>
      </A>
    </Item>
  )
}

BigButton.propTypes = {
  href: React.PropTypes.string.isRequired,
  color: React.PropTypes.string,
  children: React.PropTypes.node,
}
