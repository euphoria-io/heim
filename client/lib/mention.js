var hueHash = require('./hue-hash')

/**
 * Determines if name contains the characters in the partial name, in order.
 */
module.exports.containsSubseq = function(name, part) {
  var offset = 0
  var remain = part
  var nexdex
  while (remain !== "") {
    nexdex = name.indexOf(remain.substr(0, 1), offset)
    if (nexdex < 0) {
      return false
    }
    offset = nexdex + 1
    remain = remain.substr(1)
  }
  return true
}

/**
 * From a name and a partial produce a score. A score of
 * zero indicates "no match whatsoever", all other scores
 * are positive.
 */
module.exports.scoreMatch = function(name, part) {
  // FIXME Use proper Unicode-aware case-folding, if not already
  var part_cf = part.toLowerCase()
  var name_cf = name.toLowerCase()
  // Check prefixes, then infixes, then subsequences -- and for
  // each, try case-sensitive and then insensitive.
  if (name.startsWith(part))
    return 31
  else if (name_cf.startsWith(part_cf))
    return 30
  else if (name.contains(part))
    return 21
  else if (name_cf.contains(part_cf))
    return 20
  else if (module.exports.containsSubseq(name, part))
    return 11
  else if (module.exports.containsSubseq(name_cf, part_cf))
    return 10
  else
    return 0
}

/**
 * Given an Immutable Seq of names and a partial name, yield sorted
 * Seq of mentionable names by match relevancy (best first).
 * Mentionable names are suitable for use as mentions (do not contain
 * spaces, but do contain emoji, non-ASCII, etc.)
 */
module.exports.rankCompletions = function(names, part) {
  var partStrip = hueHash.stripSpaces(part)
  return names
    .map(function(name) {
      var stripped = hueHash.stripSpaces(name)
      return [stripped, module.exports.scoreMatch(stripped, partStrip)]
    })
    .filter(entry => entry[1] > 0)
    .sort(function(a, b) {
      return b[1] - a[1]
    })
    .map(entry => entry[0])
}
