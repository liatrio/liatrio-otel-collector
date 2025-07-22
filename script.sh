git filter-branch --env-filter '
if [[ "$GIT_AUTHOR_EMAIL" == *lululemon* ]]; then
    GIT_AUTHOR_EMAIL="jordana@liatrio.com"
    GIT_AUTHOR_NAME="Jordan Allen"
fi
if [[ "$GIT_COMMITTER_EMAIL" == *lululemon* ]]; then
    GIT_COMMITTER_EMAIL="jordana@liatrio.com"
    GIT_COMMITTER_NAME="Jordan Allen"
fi
' --tag-name-filter cat -- --all
