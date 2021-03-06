<!doctype html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-1BmE4kWBq78iYhFldvKuhfTAU6auU8tT94WrHftjDbrCEXSU1oBoqyl2QvZ6jIW3" crossorigin="anonymous">
</head>
<body>

<div class="container">
    <nav class="navbar navbar-light bg-light">
        <div class="container-fluid">
            <a class="navbar-brand">Teleporter</a>
            <span class="navbar-brand mb-0 h1">State: {{ .client.ConnectionState }}</span>
        </div>
    </nav>

    <table class="table">
    <thead>
    <th>Type</th>
    <th>Name</th>
    <th>Progress</th>
    <th>Status</th>
    <th>Details</th>
    </thead>
    {{ range $idx, $task := .client.TaskMonitor.List (Offset .request) (Limit .request) }}
    <tr>
        <th scope="row">{{$task.Type}}</th>
        <td>{{$task.Name}}</td>
        <td>
            <div class="progress">
                <div class="progress-bar" role="progressbar" style="width: {{$task.Progress}}%" aria-valuenow="{{$task.Progress}}" aria-valuemin="0" aria-valuemax="100">{{$task.Progress}}%</div>
            </div>
        </td>
        <td>{{$task.Status}}</td>
        <td>{{$task.Details}}</td>
    </tr>
    {{end}}
</table>
</div>

</body>
</html>
