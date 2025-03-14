# smolbot

A smol IRC bot

## wtf

This bot is intentionally very light on features.  The bot provides a webhook listener that only accepts JSON POST requests from `localhost` as well as an example chat command.  By design, it does not provide any plugins or modules of any sort.

This bot is intended to be customized before deployment, hence its rather bare-bones nature.  The goal is to provide a way to exfiltrate data from or provide polling-based monitoring for legacy software, that would otherwise be laborious or impractical to retrieve, via IRC.
