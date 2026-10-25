package app.stasharr.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Dns
import androidx.compose.material.icons.rounded.Lock
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.stasharr.app
import app.stasharr.data.ApiException
import app.stasharr.ui.components.GradientButton
import app.stasharr.ui.components.GradientText
import app.stasharr.ui.theme.Brand
import kotlinx.coroutines.launch

/** Adds http:// when missing, and :2704 when no port is given. */
private fun normaliseServer(input: String): String {
    var s = input.trim().trimEnd('/')
    if (s.isEmpty()) return s
    if (!s.startsWith("http://") && !s.startsWith("https://")) s = "http://$s"
    val afterScheme = s.substringAfter("://")
    val host = afterScheme.substringBefore('/')
    if (!host.contains(':') && s.startsWith("http://")) {
        val rest = afterScheme.removePrefix(host)
        s = s.substringBefore("://") + "://" + host + ":2704" + rest
    }
    return s
}

@Composable
private fun Field(
    value: String,
    onChange: (String) -> Unit,
    label: String,
    icon: ImageVector,
    keyboard: KeyboardType = KeyboardType.Text,
    password: Boolean = false,
    imeAction: ImeAction = ImeAction.Next,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        label = { Text(label) },
        leadingIcon = { Icon(icon, null) },
        singleLine = true,
        shape = RoundedCornerShape(16.dp),
        visualTransformation = if (password) PasswordVisualTransformation() else androidx.compose.ui.text.input.VisualTransformation.None,
        keyboardOptions = KeyboardOptions(keyboardType = keyboard, imeAction = imeAction),
        colors = OutlinedTextFieldDefaults.colors(
            focusedBorderColor = Brand.Pink,
            unfocusedBorderColor = Brand.Outline,
            focusedContainerColor = Brand.Surface,
            unfocusedContainerColor = Brand.Surface,
            focusedLeadingIconColor = Brand.Pink,
            cursorColor = Brand.Pink,
            focusedLabelColor = Brand.Pink,
        ),
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
fun LoginScreen(onDone: () -> Unit) {
    val scope = rememberCoroutineScope()
    var server by rememberSaveable { mutableStateOf(app.session.serverUrl ?: "") }
    var username by rememberSaveable { mutableStateOf(app.session.username ?: "") }
    var password by rememberSaveable { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    fun connect() {
        val url = normaliseServer(server)
        if (url.isEmpty()) {
            error = "Enter your server's address"
            return
        }
        busy = true
        error = null
        scope.launch {
            try {
                app.session.sessionCookie = null
                app.session.serverUrl = url
                if (username.isNotBlank()) {
                    app.api.login(username.trim(), password)
                    app.session.username = username.trim()
                    app.session.password = password
                } else {
                    app.session.username = null
                    app.session.password = null
                }
                // check it works
                app.repo.me()
                server = url
                onDone()
            } catch (e: ApiException) {
                error = if (e.unauthorized && username.isBlank()) {
                    "This server needs a username and password"
                } else {
                    e.message
                }
            } catch (e: Exception) {
                error = e.message ?: "Couldn't connect"
            } finally {
                busy = false
            }
        }
    }

    Box(
        Modifier
            .fillMaxSize()
            .background(
                Brush.radialGradient(
                    listOf(Brand.Pink.copy(alpha = 0.35f), Brand.Background),
                    radius = 1400f,
                )
            )
    ) {
        Box(
            Modifier
                .fillMaxSize()
                .background(
                    Brush.verticalGradient(listOf(Color.Transparent, Brand.Background.copy(alpha = 0.9f), Brand.Background))
                )
        )
        Column(
            Modifier
                .fillMaxSize()
                .statusBarsPadding()
                .navigationBarsPadding()
                .imePadding()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 28.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Spacer(Modifier.height(72.dp))
            Box(
                Modifier
                    .size(84.dp)
                    .clip(RoundedCornerShape(24.dp))
                    .background(Brand.Gradient),
                contentAlignment = Alignment.Center,
            ) {
                Icon(Icons.Rounded.PlayArrow, null, tint = Color.White, modifier = Modifier.size(52.dp))
            }
            Spacer(Modifier.height(20.dp))
            GradientText("stasharr", fontSize = 40.sp)
            Text(
                "Your library, anywhere in the house.",
                color = Brand.TextMuted,
                style = MaterialTheme.typography.bodyLarge,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(40.dp))

            Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
                Field(server, { server = it }, "Server address", Icons.Rounded.Dns, KeyboardType.Uri)
                Text(
                    "For example 192.168.1.20:2704 or https://stash.example.com",
                    color = Brand.TextMuted,
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.padding(start = 6.dp),
                )
                Field(username, { username = it }, "Username", Icons.Rounded.Person)
                Field(
                    password, { password = it }, "Password", Icons.Rounded.Lock,
                    keyboard = KeyboardType.Password, password = true, imeAction = ImeAction.Done,
                )
                Text(
                    "Leave these empty if your server has no login.",
                    color = Brand.TextMuted,
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.padding(start = 6.dp),
                )
            }

            if (error != null) {
                Spacer(Modifier.height(16.dp))
                Text(
                    error!!,
                    color = MaterialTheme.colorScheme.error,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth(),
                )
            }

            Spacer(Modifier.height(28.dp))
            if (busy) {
                CircularProgressIndicator(color = Brand.Pink)
            } else {
                GradientButton(
                    "Connect",
                    onClick = { connect() },
                    modifier = Modifier.fillMaxWidth(),
                    height = 54.dp,
                )
            }
            Spacer(Modifier.height(40.dp))
        }
    }
}
