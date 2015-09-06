var hueHash = require('./hue-hash')

/**
 * Determines if nick contains the characters in the partial nick, in order.
 */
module.exports.containsSubseq = function(nick, part) {
  var offset = 0
  var remain = part
  var nexdex
  while (remain !== "") {
    nexdex = nick.indexOf(remain.substr(0, 1), offset)
    if (nexdex < 0) {
      return false
    }
    offset++
    remain = remain.substr(1)
  }
  return true
}

/** From a nick and a partial produce a score. */
module.exports.scoreMatch = function(nick, part) {
  // FIXME Use proper Unicode-aware case-folding, if not already
  var part_cf = part.toLowerCase()
  var nick_cf = nick.toLowerCase()
  // Check prefixes, then infixes, then subsequences -- and for
  // each, try case-sensitive and then insensitive.
  if (nick.startsWith(part))
    return 7
  else if (nick_cf.startsWith(part_cf))
    return 6
  else if (nick.contains(part))
    return 5
  else if (nick_cf.contains(part_cf))
    return 4
  else if (containsSubseq(nick, part))
    return 3
  else if (containsSubseq(nick_cf, part_cf))
    return 2
  else
    return 1
}

/**
 * Given a seq of usernames and a partial nick, yield sorted
 * (and space-stripped) usernames by match relevancy (best first).
 */
module.exports.rankCompletions = function(nicks, part) {
  var partStrip = hueHash.stripSpaces(part)
  return nicks
    .map(hueHash.stripSpaces)
    .filter(Boolean)
    .sort(function(a, b) {
      var sa = scoreMatch(a, partStrip)
      var sb = scoreMatch(b, partStrip)
      return sb - sa
    })
}
