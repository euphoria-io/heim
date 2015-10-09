var hueHash = require('./hue-hash')

/**
 * Custom lexicographic comparator on pairs of equal-length
 * arrays. Does not do a deep comparison on sub-arrays.
 */
function compareArrays(a, b) {
  var len = a.length;
  for (var i = 0; i < len; i++) {
    var elA = a[i]
    var elB = b[i]
    if (elA < elB) {
      return -1
    } else if (elA > elB) {
      return 1
    }
    // continue if equal...
  }
  return 0;
}

/**
 * Determine if name contains the characters in the partial name, in
 * order. If found, yield the index where the subsequence starts, else
 * yield -1.
 */
module.exports.findSubseq = function(name, part) {
  // Walk the characters in partial name, skipping forward in full
  // name to match until we can't find any more matches or we finish
  // walking.
  var searchFrom = 0
  var matchStart = -1
  for (var partOffset = 0; partOffset < part.length; partOffset++) {
    var nextChar = part[partOffset]
    var foundAt = name.indexOf(nextChar, searchFrom)
    if (foundAt === -1) {
      return -1
    }
    if (partOffset === 0) {
      matchStart = foundAt
    }
    searchFrom = foundAt + 1
  }
  return matchStart
}

/**
 * Match partial against name without looking at case variants.  Yield
 * array of [contiguousScore, prefixScore, start], or null for no
 * match.
 */
module.exports.matchPinningCase = function(name, part) {
  var subseq = module.exports.findSubseq(name, part)
  if (subseq === -1) {
    return null
  }
  var infix = name.indexOf(part)

  var contiguous = infix !== -1
  var prefix = infix === 0
  var start = contiguous ? infix : subseq

  return [contiguous ? -1 : 0,
          prefix ? -1 : 0,
          start]
}

/**
 * From a name and a partial produce a score. Scores are arrays of
 * constant length and are intended to be compared lexicographically.
 * Lower values are better matches. If the result is null, there is
 * no match whatsoever.
 */
module.exports.scoreMatch = function(name, part) {
  // FIXME Use proper Unicode-aware case-folding, if not already
  var partLower = part.toLowerCase()
  var nameLower = name.toLowerCase()

  var casefoldScore = module.exports.matchPinningCase(nameLower, partLower)
  if (!casefoldScore) {
    return null;
  }
  var casekeepScore = module.exports.matchPinningCase(name, part)

  // Inject case-preservation just before last score element, then
  // choose best of the two options (if we have two options.)

  casefoldScore.splice(2, 0, 0)
  if (casekeepScore) {
    casekeepScore.splice(2, 0, -1)
    if (compareArrays(casekeepScore, casefoldScore) <= 0) {
      return casekeepScore
    }
  }
  return casefoldScore
}

/**
 * Yield {completion, score} for a pair of name, stripped
 * partial name.
 */
function annotateScore(name, partStrip) {
  var stripped = hueHash.stripSpaces(name)
  var score = module.exports.scoreMatch(stripped, partStrip)
  if (score) {
    // Add tie-breakers. We first sort by lowercased names and
    // then by the original names so that we don't get orderings
    // like ["A", "Z", "a"]. This still sorts uppercase before
    // lowercase, which is fine.
    score.push(name.toLowerCase(), name)
  }
  return {completion: stripped, score: score}
}

/**
 * Given an Immutable Seq of names and a partial name, yield sorted
 * Seq of mentionable names by match relevancy (best first).
 * Names that do not match at all are omitted from the result.
 * Mentionable names are suitable for use as mentions (do not contain
 * spaces, but do contain emoji, non-ASCII, etc.)
 */
module.exports.rankCompletions = function(names, part) {
  var partStrip = hueHash.stripSpaces(part)
  return names
    .filter(Boolean)
    .map(name => annotateScore(name, partStrip))
    .filter(entry => entry.score)
    // Use a custom lexicographic array sorter because JS's native
    // array comparison stringifies numeric elements for comparison,
    // meaning negative numbers compare incorrectly. We need negative
    // numbers in the score so that better matches come up at the
    // front -- because that needs to match the tie-breaker of
    // asciibetical ordering!
    .sortBy(entry => entry.score, compareArrays)
    .map(entry => entry.completion)
}
