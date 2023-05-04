use rhai::{Engine, Scope};

fn main() {
    let engine = Engine::new();
    let mut scope = Scope::new();
    let ast = match engine.compile_file("build.rhai".into())  {
        Ok(ast) => ast,
        Err(err) => {
            println!("compile failed: {}", err);
            return;
        },
    };
    if let Some(err) = engine.run_ast_with_scope(&mut scope, &ast).err() {
        println!("Error: {}", err);
    }
}

#[cfg(test)]
mod test {
    use rhai::FnAccess;
    use rhai::{Engine, Scope};

    #[test]
    fn test_eval() {
        let mut engine = Engine::new();
        engine.set_allow_shadowing(false);
        engine.set_allow_switch_expression(false);
        engine.on_print(|_| ());
        assert_eq!(engine.eval::<i64>("69 + 420").unwrap(), 69 + 420);
    }

    #[test]
    fn test_find_function() {
        let mut engine = Engine::new();
        engine.on_print(|_| ());
        let ast = engine
            .compile(
                r#"
        fn run() {
            print("hello there guys");
            1
        }
        1 + run()
        "#,
            )
            .unwrap();

        let funcs: Vec<_> = ast.iter_functions().collect();
        assert_eq!(funcs[0].name, "run");
        assert_eq!(funcs[0].access, FnAccess::Public);
        let ast = ast.clone_functions_only();
        let mut scope = Scope::new();
        let ret = engine.call_fn::<i64>(&mut scope, &ast, "run", ());
        match ret {
            Ok(result) => assert_eq!(1, result),
            Err(e) => panic!("{}", e),
        }
    }
}
