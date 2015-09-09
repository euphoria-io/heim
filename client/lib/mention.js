var hueHash = require('./hue-hash')

/**
 * Determines if name contains the characters in the partial name, in order.
 */
module.exports.containsSubseq = function(name, part) {
  // Walk the characters in partial name, skipping forward in full
  // name to match until we can't find any more matches or we finish
  // walking.
  var offset = 0
  for(var partOffset = 0; partOffset < part.length; partOffset++) {
    var nextChar = part[partOffset]
    offset = name.indexOf(nextChar, offset)
    if (offset === -1) {
      return false
    }
    offset++
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
  var partLowercase = part.toLowerCase()
  var nameLowercase = name.toLowerCase()
  // Check prefixes, then infixes, then subsequences -- and for
  // each, try case-sensitive and then insensitive.
  // Want something faster but uglier?
  // https://github.com/timmc/lib-1666/commit/6bd6f8a7635074f098e3d498cdd248450559b013
  var indexOfCs = name.indexOf(part);
  var indexOfCi = nameLowercase.indexOf(partLowercase);
  if (indexOfCs === 0) {
    return 31
  } else if (indexOfCi === 0) {
    return 30
  } else if (indexOfCs >= 0) {
    return 21
  } else if (indexOfCi >= 0) {
    return 20
  } else if (module.exports.containsSubseq(name, part)) {
    return 11
  } else if (module.exports.containsSubseq(nameLowercase, partLowercase)) {
    return 10
  } else {
    return 0
  }
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
    .sortBy(entry => -entry[1])
    .map(entry => entry[0])
}
