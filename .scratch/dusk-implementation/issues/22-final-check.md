# 22 — Final check: every frame enforced

**What to build:** Close out the revamp: every screen frame on pages 01–18 is enforced, nothing is pending, and nothing from the old look is left behind. Run a full pass of the frame-matching check in all modes it covers, remove the legacy colour theme and any rendering paths the new screens replaced, update DESIGN.md (the keybind map's "planned" rows are now built) and the user-facing README keys, and review the whole change against DESIGN.md.

**Blocked by:** 01 to 21 — every other ticket

**Status:** done

- [x] The ledger has no pending frames; the check passes for every frame on pages 01–18 (excluding 02/breakpoint-rule and DEV NOTE frames)
- [x] Every 00 Foundations component variant matches in dark and in the light preview
- [x] No legacy colour theme or unused rendering code remains
- [x] DESIGN.md keybind map has no "planned" rows; the README documents the current keys
- [x] Full test suite passes; board.db / board.json format unchanged from before the revamp
