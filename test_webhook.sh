#!/bin/bash

# Configuration
WEBHOOK_URL="${WEBHOOK_URL:-http://localhost:8001/webhook}"

# Load webhook secret from .env file if not already set
if [ -z "$GITHUB_WEBHOOK_SECRET" ] && [ -f ".env" ]; then
    export $(grep GITHUB_WEBHOOK_SECRET .env | xargs)
fi

WEBHOOK_SECRET="${GITHUB_WEBHOOK_SECRET:-your-webhook-secret-here}"

# The payload (GitHub push event)
PAYLOAD='{
  "ref": "refs/heads/feature/performance-issues",
  "before": "0000000000000000000000000000000000000000",
  "after": "f67f195b0fe15c8b271f346c90a66c483e01fde6",
  "repository": {
    "id": 1151939873,
    "node_id": "R_kgDORKk1IQ",
    "name": "af-demo-repo",
    "full_name": "yongchenglow/af-demo-repo",
    "private": false,
    "owner": {
      "name": "yongchenglow",
      "email": "lowyongcheng@hotmail.com",
      "login": "yongchenglow",
      "id": 19281905,
      "node_id": "MDQ6VXNlcjE5MjgxOTA1",
      "avatar_url": "https://avatars.githubusercontent.com/u/19281905?v=4",
      "gravatar_id": "",
      "url": "https://api.github.com/users/yongchenglow",
      "html_url": "https://github.com/yongchenglow",
      "followers_url": "https://api.github.com/users/yongchenglow/followers",
      "following_url": "https://api.github.com/users/yongchenglow/following{/other_user}",
      "gists_url": "https://api.github.com/users/yongchenglow/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/yongchenglow/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/yongchenglow/subscriptions",
      "organizations_url": "https://api.github.com/users/yongchenglow/orgs",
      "repos_url": "https://api.github.com/users/yongchenglow/repos",
      "events_url": "https://api.github.com/users/yongchenglow/events{/privacy}",
      "received_events_url": "https://api.github.com/users/yongchenglow/received_events",
      "type": "User",
      "user_view_type": "public",
      "site_admin": false
    },
    "html_url": "https://github.com/yongchenglow/af-demo-repo",
    "description": null,
    "fork": false,
    "url": "https://api.github.com/repos/yongchenglow/af-demo-repo",
    "forks_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/forks",
    "keys_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/keys{/key_id}",
    "collaborators_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/collaborators{/collaborator}",
    "teams_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/teams",
    "hooks_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/hooks",
    "issue_events_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/issues/events{/number}",
    "events_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/events",
    "assignees_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/assignees{/user}",
    "branches_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/branches{/branch}",
    "tags_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/tags",
    "blobs_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/git/blobs{/sha}",
    "git_tags_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/git/tags{/sha}",
    "git_refs_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/git/refs{/sha}",
    "trees_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/git/trees{/sha}",
    "statuses_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/statuses/{sha}",
    "languages_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/languages",
    "stargazers_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/stargazers",
    "contributors_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/contributors",
    "subscribers_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/subscribers",
    "subscription_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/subscription",
    "commits_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/commits{/sha}",
    "git_commits_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/git/commits{/sha}",
    "comments_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/comments{/number}",
    "issue_comment_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/issues/comments{/number}",
    "contents_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/contents/{+path}",
    "compare_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/compare/{base}...{head}",
    "merges_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/merges",
    "archive_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/{archive_format}{/ref}",
    "downloads_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/downloads",
    "issues_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/issues{/number}",
    "pulls_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/pulls{/number}",
    "milestones_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/milestones{/number}",
    "notifications_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/notifications{?since,all,participating}",
    "labels_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/labels{/name}",
    "releases_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/releases{/id}",
    "deployments_url": "https://api.github.com/repos/yongchenglow/af-demo-repo/deployments",
    "created_at": 1770442224,
    "updated_at": "2026-02-07T06:22:53Z",
    "pushed_at": 1770448422,
    "git_url": "git://github.com/yongchenglow/af-demo-repo.git",
    "ssh_url": "git@github.com:yongchenglow/af-demo-repo.git",
    "clone_url": "https://github.com/yongchenglow/af-demo-repo.git",
    "svn_url": "https://github.com/yongchenglow/af-demo-repo",
    "homepage": null,
    "size": 17,
    "stargazers_count": 0,
    "watchers_count": 0,
    "language": "Python",
    "has_issues": true,
    "has_projects": true,
    "has_downloads": true,
    "has_wiki": true,
    "has_pages": false,
    "has_discussions": false,
    "forks_count": 0,
    "mirror_url": null,
    "archived": false,
    "disabled": false,
    "open_issues_count": 1,
    "license": null,
    "allow_forking": true,
    "is_template": false,
    "web_commit_signoff_required": false,
    "topics": [],
    "visibility": "public",
    "forks": 0,
    "open_issues": 1,
    "watchers": 0,
    "default_branch": "main",
    "stargazers": 0,
    "master_branch": "main"
  },
  "pusher": {
    "name": "yongchenglow",
    "email": "lowyongcheng@hotmail.com"
  },
  "forced": false,
  "sender": {
    "login": "yongchenglow",
    "id": 19281905,
    "node_id": "MDQ6VXNlcjE5MjgxOTA1",
    "avatar_url": "https://avatars.githubusercontent.com/u/19281905?v=4",
    "gravatar_id": "",
    "url": "https://api.github.com/users/yongchenglow",
    "html_url": "https://github.com/yongchenglow",
    "followers_url": "https://api.github.com/users/yongchenglow/followers",
    "following_url": "https://api.github.com/users/yongchenglow/following{/other_user}",
    "gists_url": "https://api.github.com/users/yongchenglow/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/yongchenglow/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/yongchenglow/subscriptions",
    "organizations_url": "https://api.github.com/users/yongchenglow/orgs",
    "repos_url": "https://api.github.com/users/yongchenglow/repos",
    "events_url": "https://api.github.com/users/yongchenglow/events{/privacy}",
    "received_events_url": "https://api.github.com/users/yongchenglow/received_events",
    "type": "User",
    "user_view_type": "public",
    "site_admin": false
  },
  "created": true,
  "deleted": false,
  "base_ref": null,
  "compare": "https://github.com/yongchenglow/af-demo-repo/commit/f67f195b0fe1",
  "commits": [
    {
      "id": "f67f195b0fe15c8b271f346c90a66c483e01fde6",
      "tree_id": "b2b07a8b835d7d03f29cf317090dad22a1f3eeda",
      "distinct": true,
      "message": "Add payment processing with credit card support",
      "timestamp": "2026-02-07T13:59:33+08:00",
      "url": "https://github.com/yongchenglow/af-demo-repo/commit/f67f195b0fe15c8b271f346c90a66c483e01fde6",
      "author": {
        "name": "Yong Cheng Low",
        "email": "lowyongcheng@hotmail.com",
        "date": "2026-02-07T13:59:33+08:00",
        "username": "yongchenglow"
      },
      "committer": {
        "name": "Yong Cheng Low",
        "email": "lowyongcheng@hotmail.com",
        "date": "2026-02-07T13:59:33+08:00",
        "username": "yongchenglow"
      },
      "added": ["payments.py"],
      "removed": [],
      "modified": []
    }
  ],
  "head_commit": {
    "id": "f67f195b0fe15c8b271f346c90a66c483e01fde6",
    "tree_id": "b2b07a8b835d7d03f29cf317090dad22a1f3eeda",
    "distinct": true,
    "message": "Add payment processing with credit card support",
    "timestamp": "2026-02-07T13:59:33+08:00",
    "url": "https://github.com/yongchenglow/af-demo-repo/commit/f67f195b0fe15c8b271f346c90a66c483e01fde6",
    "author": {
      "name": "Yong Cheng Low",
      "email": "lowyongcheng@hotmail.com",
      "date": "2026-02-07T13:59:33+08:00",
      "username": "yongchenglow"
    },
    "committer": {
      "name": "Yong Cheng Low",
      "email": "lowyongcheng@hotmail.com",
      "date": "2026-02-07T13:59:33+08:00",
      "username": "yongchenglow"
    },
    "added": ["payments.py"],
    "removed": [],
    "modified": []
  }
}'

# Generate HMAC SHA256 signature
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" | sed 's/^.* //')

echo "Sending webhook to $WEBHOOK_URL"
echo "Event Type: push"
echo "Signature: sha256=$SIGNATURE"
echo ""

# Send the webhook request
curl -X POST "$WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
  -H "X-GitHub-Delivery: $(uuidgen)" \
  -d "$PAYLOAD" \
  -v

echo ""
echo "Webhook sent successfully!"
