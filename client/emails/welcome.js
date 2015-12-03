import React from 'react'

import { Item, Text, Link } from './email'
import { StandardEmail, TopBubbleBox, BigButton, BodyBox, Footer, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo.png">
      <Item align="center">
        <Text {...textDefaults} fontSize={52}>Hi!</Text>
      </Item>
      <Item align="center">
        <Text {...textDefaults} fontSize={18} color="#9f9f9f">Welcome to {'{{.SiteName}}'} :)</Text>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item align="center">
        <Text {...textDefaults}>Your account is almost ready:</Text>
      </Item>
      <BigButton color="#80c080" href="{{.VerifyEmailURL}}">
        verify your email address
      </BigButton>
      <Item>
        <Text {...textDefaults}>We hope you have a wonderful time on <Link {...textDefaults} href="{{.SiteURL}}">{'{{.SiteName}}'}</Link>. If you have any questions or comments, feel free to <Link {...textDefaults} href="mailto:{{.HelpAddress}}">contact us</Link>.</Text>
      </Item>
    </BodyBox>
    <Footer>
      <Text {...textDefaults} fontSize={13} color="#7d7d7d">This message was sent to <Link {...textDefaults} textDecoration="none" href="mailto:{{.AccountEmailAddress}}">{'{{.AccountEmailAddress}}'}</Link> because someone signed up for an account on <Link {...textDefaults} textDecoration="none" href="{{.SiteURL}}">{'{{.SiteURLShort}}'}</Link> with this email address. If you did not request this email, please disregard.</Text>
    </Footer>
  </StandardEmail>
)
