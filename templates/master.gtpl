{{define "header"}}
<html>

<head>
    <title>{{.Title}}</title>
    <meta charset="utf-8" />
    <link rel="stylesheet" href="/static/jquery-ui-1.11.4/jquery-ui.min.css">
    <link rel="stylesheet" href="/static/bootstrap-3.3.6/css/bootstrap.min.css">
    <link rel="stylesheet" href="/static/bootstrap-3.3.6/css/bootstrap-theme.min.css">
    <link rel="stylesheet" href="/static/css/style.css">
</head>
<body>
{{end}}
{{define "nav"}}
<!-- Fixed navbar -->
<nav class="navbar navbar-default navbar-fixed-top">
  <div class="container">
    <div class="navbar-header">
      <a class="navbar-brand" href="/category">Project</a>
    </div>
    <div id="navbar" class="navbar-collapse collapse">
      <ul class="nav navbar-nav">
        <li name="cate"><a href="/category">Category</a></li>
        <li name="item"><a href="/item">Item</a></li>
        <li name="user"><a href="/user">User</a></li>
      </ul>
    </div><!--/.nav-collapse -->
  </div>
</nav>
{{end}}
{{define "footer"}}
<script src="/static/js/jquery-1.12.3.min.js"></script>
<script src="/static/js/js.cookie-2.1.1.min.js"></script>
<script src="/static/jquery-ui-1.11.4/jquery-ui.min.js"></script>
<script src="/static/bootstrap-3.3.6/js/bootstrap.min.js"></script>
<script src="/static/js/common.js"></script>
</body>

</html>
{{end}}
