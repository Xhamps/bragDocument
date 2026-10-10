# NotificationItem

One activity row: avatar, the person's name over what they did, and a relative time.

- **Provide** `name`, `action` ("invited you to a chat"), `time` ("5m ago"), an `avatar` photo URL (initials otherwise) and `unread` for a blue dot.
- Stack rows with `space-2` gaps inside a Card titled "Notifications", with a footer of "2 unread" and a glass "Mark all as read" button.
- Keep `action` to one line; sentence case, no trailing period.
