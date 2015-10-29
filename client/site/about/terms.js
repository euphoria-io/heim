import fs from 'fs'
import React from 'react'

import { MainPage, PolicyNav, Markdown } from '../common'


module.exports = (
  <MainPage title="euphoria: terms of service" nav={<PolicyNav selected="terms" />}>
    <Markdown className="policy" content={fs.readFileSync(__dirname + '/terms.md', 'utf8')} />
  </MainPage>
)
