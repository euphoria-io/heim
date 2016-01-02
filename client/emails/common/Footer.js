import React from 'react'

import { Item } from 'react-html-email'


export default function Footer(props) {
  return (
    <Item style={{paddingLeft: '20px', paddingRight: '20px', paddingTop: '12px'}}>
      {props.children}
    </Item>
  )
}

Footer.propTypes = {
  children: React.PropTypes.node,
}
