<!doctype html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-1BmE4kWBq78iYhFldvKuhfTAU6auU8tT94WrHftjDbrCEXSU1oBoqyl2QvZ6jIW3" crossorigin="anonymous">
</head>
<body>
<div class="container">
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
        <td><progress max="100" value="{{$task.Progress}}"></progress></td>
        <td>{{$task.Status}}</td>
        <td>{{$task.Details}}</td>
    </tr>
    {{end}}
</table>
</div>

</body>
</html>
