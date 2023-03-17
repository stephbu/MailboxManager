# Mailbox Manager

## Purpose

Emulate some neat Outlook email features that gmail sorely lacks
- Keep Last 'n'
- Retention 'n' Days

## How It Works
Create custom labels, then create filters to apply labels to emails as they arrive

Expected format of labels is as follows:
```
- ManagedLabels/KeepDaysX
- ManagedLabels/KeepLastX
```
For example, to keep the last 10 mails from a specific sender,
you'd apply the label called:

```ManagedLabels/KeepLast10```

Or to keep a mail in your inbox for up to a week before being deleted,
you'd apply a label called

```ManagedLabels/KeepDays7```

Each time you run the code, it will comb through recognized managed
labels, identifying mails that should be retained, and those out of retention scope.
Those out of scope will be deleted.

## Permissions

It requires full ```mail.google.com``` access permissions in order to delete mails.

## Getting Started

### Code
You'll need some Go Tools, VSCode and Goland are my current favorites.
Clone this repo somewhere convenient.

### Secrets
You'll need to provide a set of secrets in order to get started with the Gmail API.
The code needs a set of credentials and a token stored in the secrets folder.
The secrets folder is currently not committed to this repo, and should not be committed.

### Secrets Folder
The code assumes that secrets are stored in
```
<code>/secrets/
```

#### Credentials File
1. Sign into the Google Cloud API dashboard using your Gmail account
2. Create a project called "MailboxManager" in the Google Cloud Platform website.
3. Create an OAuth 2.0 credential in the API Dashboard
https://console.cloud.google.com/apis/dashboard
4. Download this credential and store it in ```<code>/secrets/credentials.json```

#### Token
Assuming that the credentials file exists, this token is generated when the application is first run,
copy and paste the full URL from the console into a browser.  You will be prompted to log in, accept the access requested, in return
you will get a token string.  Copy and paste this token string into the running application console, and press enter to save the token.
(written to ```<code>/secrets/token.json```)

This token **will** periodically expire.  Delete the file, re-initiate the process.
