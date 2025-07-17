# yamlot

# TODO

- [ ] Scan comments: Add TokenComment, treat # as the start of a comment until newline, and decide whether to preserve or discard based on context.
- [ ] Scan doc end: Detect ... at column 1 with trailing whitespace or newline; emit TokenDocEnd.
- [ ] Recognize more tokens: Gradually introduce support for quoted scalars, anchors, tags, block indicators (|, >) and mapping keys (:) in separate substates.
