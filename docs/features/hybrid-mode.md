# Hybrid Mode

Combine manual commands and AI tasks freely in the same session. Take control when you need precision, delegate to AI when you want automation.

## How It Works

No mode switch needed. Nullhand always understands both:

- Messages starting with `/` → execute directly (manual)
- Any other message → send to AI (agent)

## Example Session

```
You:  Open Safari
Bot:  ✓ Safari opened

You:  /screenshot
Bot:  [screenshot — you see Safari]

You:  Navigate to my server panel at https://panel.example.com
Bot:  Navigating...
      [screenshot — login page]

You:  /type admin@example.com
Bot:  ✓

You:  /key tab
Bot:  ✓

You:  /type mypassword
Bot:  ✓

You:  /key enter
Bot:  ✓

You:  Find the SSL section and renew the cert for example.com
Bot:  [screenshot — AI sees the dashboard]
      Found SSL section. Clicking...
      [screenshot — SSL page]
      Certificate expires in 3 days. Renewing...
      Done. New expiry: 2027-04-11 ✓
```

## When to Use Each Mode

**Use manual when:**
- You need to click a specific pixel
- The AI got stuck on something simple
- You want zero latency, zero API cost
- You are entering passwords or sensitive data

**Use AI when:**
- The task has many steps
- You need the AI to read and react to screen content
- You want to describe intent rather than specific actions

**Intervene mid-task:**
```
Bot:  I cannot find the SSL button...
You:  /screenshot
Bot:  [screenshot]
You:  /click 612 380
Bot:  ✓
You:  Now continue from here
Bot:  Got it. I can see the SSL panel now...
```

## Tips

1. Use `/screenshot` frequently — it is your eyes on the machine
2. Manual commands cost zero API credits
3. Use `/stop` if the AI goes off track, then guide it manually
4. Enter credentials manually with `/type` — do not pass them to the AI
