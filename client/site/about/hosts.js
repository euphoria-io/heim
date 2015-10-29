import fs from 'fs'
import React from 'react'

import { MainPage, PolicyNav, Markdown } from '../common'


module.exports = (
  <MainPage title="euphoria: room host policy" nav={<PolicyNav selected="hosts" />}>
    <Markdown className="policy" content={fs.readFileSync(__dirname + '/hosts.md', 'utf8')} />
  </MainPage>
)
