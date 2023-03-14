# Cachin

Cachin is a package for functional caching and memoization in go.

## cache
The cache package provides functionality for caching function results.
You can use these functions to cache function results both in memory as-well-as in an external data store.

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

## memoize
The memoization package provides functionality for memoizing function results.
You can use these functions to cache function results both in memory as-well-as in an external data store.
Additionally, this cache is set based on the input parameter so different inputs will have their own individual cache.

Example 1: cache function results in memory. This makes repeatedly calling the GetTeams function much faster since
only the first call will result in a network call.
```go
// GetTeam gets a team from an external api. The results will be cached in memory for at least one hour.
// The cached data is tied to the input parameter such that calls with different inputs will have their own
// individual cache.
var GetTeam = memoize.InMemory(time.Hour, func(ctx context.Context, teamName string) (*Team, error) {
    client := &http.Client{}
    resp, err := client.Get("https://api.weavedev.net/team/" + teamName)
    if err != nil {
        return nil, err
    }

    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    team := &Team{}
    err = json.Unmarshal(body, team)
    if err != nil {
        return nil, err
    }

    return team, nil
})
```

Example 2: cache function results in memory and on disk.
Like example 1 this improves performance.
It also allows the cache to be restored across runs which can be useful for short-lived process like cron jobs or cli tools
```go
// GetTeam gets a team from an external api. The results will be cached in memory for at least one hour.
// The cached data is tied to the input parameter such that calls with different inputs will have their own
// individual cache. Additionally, the cache will be backed by the file system so it can be restored between program runs
var GetTeam = memoize.OnDisk(filepath.Join("cache", "team"), time.Hour, func(ctx context.Context, teamName string) (*Team, error) {
    client := &http.Client{}
    resp, err := client.Get("https://api.weavedev.net/team/" + teamName)
    if err != nil {
        return nil, err
    }

    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    team := &Team{}
    err = json.Unmarshal(body, team)
    if err != nil {
        return nil, err
    }

    return team, nil
})
```
