# Cachin

Cachin is a package for functional caching and memoization in Go.

To fetch, build and install from the GitHub source:

```sh
go get github.com/weave-lab/cachin
```

## cache
The cache package provides functionality for caching function results.
You can use these functions to cache function results both in memory and in an external data store.
Be aware that if you are caching large amounts of data with long TTLs you may run into OOM issues.

It's important to note that the cached function may return expired data.
This can happen when your cached function returns an error but the previous cache value still exists.
In this case valid cache data will be returned along with your function's error.
As the developer it is up to you to determine if this stale data is safe to use or if it should be ignored.

Example 1: cache function results in memory. This makes repeatedly calling the GetTeams function much faster since
only the first call will result in a network call.
```go
// GetTeams gets a list of teams from an external api. The results will be cached in memory for 
// at least one hour
var GetTeams = cache.InMemory(time.Hour, func(ctx context.Context) ([]Team, error) {
    client := &http.Client{}
    resp, err := client.Get("https://api.weavedev.net/teams")
    if err != nil {
        return nil, err
    }

    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var teams []Team
    err = json.Unmarshal(body, &teams)
    if err != nil {
        return nil, err
    }

    return teams, nil
})
```

Example 2: cache function results in memory and on disk.
Like example 1 this improves performance.
It also allows the cache to be restored across runs which can be useful for short-lived process like cron jobs or cli tools
```go
// GetTeams gets a list of teams from an external api. The results will be cached in memory for at least one hour.
// Additionally, the cache will be backed by the file system so it can be restored between program runs
var GetTeams = cache.OnDisk(filepath.Join("cache", "teams"), time.Hour, func(ctx context.Context) ([]Team, error) {
    client := &http.Client{}
    resp, err := client.Get("https://api.weavedev.net/teams")
    if err != nil {
        return nil, err
    }

    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var teams []Team
    err = json.Unmarshal(body, &teams)
    if err != nil {
        return nil, err
    }

    return teams, nil
})
```
