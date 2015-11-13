import React from 'react'

import { Item } from '../email'


export default function Footer(props) {
  return (
    <Item style={{paddingLeft: '20px', paddingRight: '20px', paddingTop: '20px'}}>
      {props.children}
    </Item>
  )
}

Footer.propTypes = {
  children: React.PropTypes.node,
}
